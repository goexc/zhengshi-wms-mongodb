export interface ImagesRequest {
  page: number;
  size: number;
  name?: string;
}

export interface ImagesResponse {
  code: number;
  msg: string;
  data: ImagePaginate;
}

export interface ImagePaginate{
  total: number;
  list: ImageItem[];
}

export interface ImageItem {
  url: string;
  alt: string;
}

export interface ImageRequest{

}